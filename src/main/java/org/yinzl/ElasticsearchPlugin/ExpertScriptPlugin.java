package org.yinzl.ElasticsearchPlugin;


import java.io.IOException;
import java.nio.ByteBuffer;
import java.util.Base64;
import java.util.Collection;
import java.util.Map;
import java.util.Set;

import org.apache.lucene.index.LeafReaderContext;
import org.elasticsearch.common.settings.Settings;
import org.elasticsearch.plugins.Plugin;
import org.elasticsearch.plugins.ScriptPlugin;
import org.elasticsearch.script.ScoreScript;
import org.elasticsearch.script.ScoreScript.LeafFactory;
import org.elasticsearch.script.ScriptContext;
import org.elasticsearch.script.ScriptEngine;
import org.elasticsearch.script.ScriptFactory;
import org.elasticsearch.search.lookup.SearchLookup;
import org.roaringbitmap.RoaringBitmap;

/**
 * An example script plugin that adds a {@link ScriptEngine}
 * implementing expert scoring.
 */
public class ExpertScriptPlugin extends Plugin implements ScriptPlugin {

    @Override
    public ScriptEngine getScriptEngine(
        Settings settings,
        Collection<ScriptContext<?>> contexts
    ) {
        return new MyExpertScriptEngine();
    }

    /** An example {@link ScriptEngine} that uses Lucene segment details to
     *  implement pure document frequency scoring. */
    // tag::expert_engine
    private static class MyExpertScriptEngine implements ScriptEngine {
        @Override
        public String getType() {
            return "expert_scripts";
        }

        @Override
        public <T> T compile(
            String scriptName,
            String scriptSource,
            ScriptContext<T> context,
            Map<String, String> params
        ) {
            if (context.equals(ScoreScript.CONTEXT) == false) {
                throw new IllegalArgumentException(getType()
                        + " scripts cannot be used for context ["
                        + context.name + "]");
            }
            // we use the script "source" as the script identifier
            if ("skip_list".equals(scriptSource)) {
                ScoreScript.Factory factory = new SkipListFactory();
                return context.factoryClazz.cast(factory);
            }
            throw new IllegalArgumentException("Unknown script name "
                    + scriptSource);
        }

        @Override
        public void close() {
            // optionally close resources
        }

        @Override
        public Set<ScriptContext<?>> getSupportedContexts() {
            return Set.of(ScoreScript.CONTEXT);
        }

        private static class SkipListFactory implements ScoreScript.Factory,
                                                      ScriptFactory {
            @Override
            public boolean isResultDeterministic() {
                // PureDfLeafFactory only uses deterministic APIs, this
                // implies the results are cacheable.
                return true;
            }

            @Override
            public LeafFactory newFactory(
                Map<String, Object> params,
                SearchLookup lookup
            ) {
                return new SkipListLeafFactory(params, lookup);
            }
        }

        private static class SkipListLeafFactory implements LeafFactory {
            private final Map<String, Object> params;
            private final SearchLookup lookup;
            private RoaringBitmap bp;
            private String fieldName;
            
            private SkipListLeafFactory(
                        Map<String, Object> params, SearchLookup lookup) {
                if (params.containsKey("skip") == false) {
                    throw new IllegalArgumentException(
                            "Missing parameter [skip]");
                }
                if (params.containsKey("field_name") == false) {
                    throw new IllegalArgumentException(
                            "Missing parameter [field_name]");
                }

                
                this.params = params;
                this.lookup = lookup;
                
                String skip = params.get("skip").toString();
    			byte[] bt = Base64.getDecoder().decode(skip);
        		ByteBuffer buffer = ByteBuffer.wrap(bt);
        		this.bp = new RoaringBitmap();
    			try {
					bp.deserialize(buffer);
				} catch (IOException e) {
					// TODO Auto-generated catch block
					e.printStackTrace();
					throw new IllegalArgumentException("bitmap deserialize failed: "+ e.getMessage());
				}
    			
    			this.fieldName = params.get("field_name").toString();
            }

            @Override
            public boolean needs_score() {
                return false;  // Return true if the script needs the score
            }

            @Override
            public ScoreScript newInstance(LeafReaderContext context)
                    throws IOException {
                return new ScoreScript(params, lookup, context) {
                    
                    @Override
                    public double execute(ExplanationHolder explanation) {
                    	
                    	try {
                    		String id = this.getDoc().get(fieldName).get(0).toString();
                        	
                        	if (bp.contains(Integer.parseInt(id))) {
                        		return 0.0d;
                        	}
    						return 1d;
                    	} catch(Exception e) {
                    		return 1d;
                    	}
                        
                    }
                };
            }
            
        }
    }
    // end::expert_engine
}
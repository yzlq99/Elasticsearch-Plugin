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
public class SkipList extends Plugin implements ScriptPlugin {

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
            return "skip_list";
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
            if ("pure_df".equals(scriptSource)) {
                ScoreScript.Factory factory = new PureDfFactory();
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

        private static class PureDfFactory implements ScoreScript.Factory,
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
                return new PureDfLeafFactory(params, lookup);
            }
        }

        private static class PureDfLeafFactory implements LeafFactory {
            private final Map<String, Object> params;
            private final SearchLookup lookup;
            private final String skip;
            
            private PureDfLeafFactory(
                        Map<String, Object> params, SearchLookup lookup) {
                if (params.containsKey("skip") == false) {
                    throw new IllegalArgumentException(
                            "Missing parameter [skip]");
                }

                this.params = params;
                this.lookup = lookup;
                skip = params.get("skip").toString();
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
                    	
                    	String id = this.getDoc().get("kw.id").get(0).toString();
                    	System.out.println("kw.id: "+id);
                    	
                    	if (checkBitMap(id)) {
                    		return 0.0d;
                    	}
						return 10d;
                        
                    }
                };
            }
            
            private boolean checkBitMap(String id) {
            	
        		try {
        			byte[] bt = Base64.getDecoder().decode(skip);
            		ByteBuffer buffer = ByteBuffer.wrap(bt);
            		RoaringBitmap bp = new RoaringBitmap();
        			bp.deserialize(buffer);
        			
        			if (bp.contains(Integer.parseInt(id))) {
        				return true;
        			}
        		} catch (IOException ioe) {
        			System.out.println(ioe.getMessage());
        			return false;
        		}
            	return false;
            }
        }
    }
    // end::expert_engine
}
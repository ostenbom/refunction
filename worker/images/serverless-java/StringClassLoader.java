import java.util.Base64;

public class StringClassLoader extends ClassLoader {
    private String classString;
    public StringClassLoader(String classString) {
        super();
        this.classString = classString;
    }

    @Override
    public Class<?> findClass(String name) {
        byte[] bt = Base64.getDecoder().decode(this.classString);
        return defineClass(name, bt, 0, bt.length);
    }

}
